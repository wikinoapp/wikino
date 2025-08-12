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

  scope :by_space, ->(space_id) { where(space_id:) }
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

    blob.url(expires_in:)
  rescue => e
    Rails.logger.error("Failed to generate signed URL for attachment #{id}: #{e.message}")
    nil
  end

  # ユーザーがこの添付ファイルを閲覧可能かチェック
  sig { params(user_record: T.nilable(UserRecord)).returns(T::Boolean) }
  def viewable_by?(user_record:)
    # 添付ファイルを参照している全てのページが公開トピックかチェック
    if all_referencing_pages_public?
      # 全て公開トピックの場合は誰でもアクセス可能
      true
    elsif user_record.nil?
      # ログインしていない場合はアクセス不可
      false
    else
      # スペースメンバーかどうかをチェック
      is_space_member?(user_record:)
    end
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

  # ユーザーがスペースメンバーかチェック
  sig { params(user_record: UserRecord).returns(T::Boolean) }
  private def is_space_member?(user_record:)
    return false unless space_record

    SpaceMemberRecord.exists?(
      space_id: space_record.not_nil!.id,
      user_id: user_record.id,
      active: true
    )
  end
end
