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

  # 署名付きURLを生成（権限チェック付き）
  sig do
    params(
      space_member_record: T.nilable(SpaceMemberRecord),
      expires_in: ActiveSupport::Duration
    ).returns(T.nilable(String))
  end
  def generate_signed_url(space_member_record:, expires_in: 1.hour)
    # ポリシーを使用してアクセス権限を確認
    policy = SpaceMemberPolicy.new(
      user_record: space_member_record&.user_record,
      space_member_record:
    )

    return nil unless policy.can_view_attachment?(attachment_record: self)

    blob = blob_record
    return nil unless blob

    blob.url(expires_in:)
  rescue => e
    Rails.logger.error("Failed to generate signed URL for attachment #{id}: #{e.message}")
    nil
  end
end
