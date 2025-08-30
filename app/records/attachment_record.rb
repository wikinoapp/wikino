# typed: strict
# frozen_string_literal: true

class AttachmentRecord < ApplicationRecord
  self.table_name = "attachments"

  enum :processing_status, {
    AttachmentProcessingStatus::Pending.serialize => 0,
    AttachmentProcessingStatus::Processing.serialize => 1,
    AttachmentProcessingStatus::Completed.serialize => 2,
    AttachmentProcessingStatus::Failed.serialize => 3
  }, prefix: true

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :active_storage_attachment_record,
    class_name: "ActiveStorage::Attachment",
    foreign_key: :active_storage_attachment_id
  belongs_to :attached_space_member_record,
    class_name: "SpaceMemberRecord",
    foreign_key: :attached_space_member_id

  has_many :page_attachment_reference_records,
    class_name: "PageAttachmentReferenceRecord",
    foreign_key: :attachment_id,
    dependent: :restrict_with_exception

  scope :by_space_member, ->(attached_space_member_id) { where(attached_space_member_id:) }
  scope :recent, -> { order(attached_at: :desc) }

  # Active Storageのblobへのショートカット
  sig { returns(T.nilable(ActiveStorage::Blob)) }
  def blob_record
    active_storage_attachment_record&.blob
  end

  # ファイル名を取得
  sig { returns(T.nilable(String)) }
  def filename
    blob_record&.filename&.to_s
  end

  # コンテントタイプを取得
  sig { returns(T.nilable(String)) }
  def content_type
    blob_record&.content_type
  end

  # ファイルサイズを取得（バイト単位）
  sig { returns(T.nilable(Integer)) }
  def byte_size
    blob_record&.byte_size
  end

  # SVGファイルのサニタイズ処理
  sig { returns(T::Boolean) }
  def sanitize_svg_content
    blob = blob_record
    return false unless blob
    return false unless blob.content_type == "image/svg+xml"

    begin
      # ファイルの内容を取得
      svg_content = blob.download

      # サニタイズ処理
      sanitized_content = SvgSanitizer.sanitize(svg_content)

      # サニタイズ済みのコンテンツをアップロード
      blob.upload(StringIO.new(sanitized_content))

      true
    rescue => e
      Rails.logger.error("SVG sanitization failed: #{e.message}")
      false
    end
  end

  # 署名付きURLを生成（権限チェックは行わない - /attachments/:id エンドポイントで実施）
  sig do
    params(
      space_member_record: T.nilable(SpaceMemberRecord),
      expires_in: ActiveSupport::Duration
    ).returns(T.nilable(String))
  end
  def generate_signed_url(space_member_record:, expires_in: 1.hour)
    # 権限チェックは /attachments/:id エンドポイントで行うため、ここでは行わない
    # space_member_recordは将来の拡張用に残している

    blob = blob_record
    return nil unless blob

    # 署名付きURLを生成
    blob.url(
      expires_in:,
      # `content_type` を指定しないと、SVGファイルのContent-Typeが `application/octet-stream` になり
      # ブラウザでの表示が正しく行われない
      content_type: blob.content_type
    )
  rescue => e
    Rails.logger.error("Failed to generate signed URL for attachment #{id}: #{e.message}")
    nil
  end

  # この添付ファイルを参照している全てのページが公開トピックかチェック
  sig { returns(T::Boolean) }
  def all_referencing_pages_public?
    referencing_topics.any? && referencing_topics.all?(&:visibility_public?)
  end

  # 参照しているページのトピックを取得
  sig { returns(T::Array[TopicRecord]) }
  def referencing_topics
    @referencing_topics ||= T.let(
      page_attachment_reference_records
        .preload(page_record: :topic_record)
        .filter_map { |ref| ref.page_record&.topic_record },
      T.nilable(T::Array[TopicRecord])
    )
  end

  # リダイレクト用のURLを取得
  sig { returns(T.nilable(String)) }
  def redirect_url
    active_storage_attachment_record&.blob&.url
  end

  # 画像ファイルかどうかチェック
  sig { returns(T::Boolean) }
  def image?
    blob_record&.image? || false
  end

  # サムネイル用のvariantを取得
  sig { params(size: AttachmentThumbnailSize).returns(T.nilable(T.any(ActiveStorage::Variant, ActiveStorage::VariantWithRecord))) }
  def thumbnail_variant(size:)
    blob = blob_record
    return nil unless blob
    return nil unless blob.image?
    return nil unless blob.variable?

    variant_options = case size
    when AttachmentThumbnailSize::Card
      # ページ一覧のカード用（高解像度対応で400x400px）
      {resize_to_limit: [400, 400]}
    when AttachmentThumbnailSize::Og
      # OGP画像用（推奨: 1200x630）
      {resize_to_fit: [1200, 630]}
    else
      T.absurd(size)
    end

    blob.variant(**variant_options)
  end

  # サムネイルURLを取得
  sig { params(size: AttachmentThumbnailSize, expires_in: ActiveSupport::Duration).returns(T.nilable(String)) }
  def thumbnail_url(size:, expires_in: 1.hour)
    variant = thumbnail_variant(size:)
    return nil unless variant

    variant.processed.url(expires_in:)
  rescue => e
    Rails.logger.error("Failed to generate thumbnail URL for attachment #{id}: #{e.class.name}: #{e.message}")
    nil
  end

  # サムネイルを事前生成（廃止予定 - Active Storageが必要に応じて自動生成するため）
  # @deprecated このメソッドは使用しないでください
  sig { returns(T::Boolean) }
  def generate_thumbnails
    # 事前生成は行わない - Active Storageのon-demandで対応
    true
  end
end
