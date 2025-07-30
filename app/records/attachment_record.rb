# typed: strict
# frozen_string_literal: true

class AttachmentRecord < ApplicationRecord
  self.table_name = "attachments"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :attached_user_record, foreign_key: :attached_user_id
  belongs_to :active_storage_attachment_record,
    class_name: "ActiveStorage::Attachment"

  has_many :page_attachment_reference_records,
    class_name: "PageAttachmentReferenceRecord",
    foreign_key: :attachment_id,
    dependent: :restrict_with_exception

  scope :by_space, ->(space_database_id) { where(space_id: space_database_id) }
  scope :by_user, ->(user_database_id) { where(attached_user_id: user_database_id) }
  scope :recent, -> { order(attached_at: :desc) }

  # Active Storageのblobへのショートカット
  sig { returns(T.nilable(ActiveStorage::Blob)) }
  def blob
    active_storage_attachment&.blob
  end

  # ファイル名を取得
  sig { returns(T.nilable(String)) }
  def filename
    blob&.filename&.to_s
  end

  # コンテントタイプを取得
  sig { returns(T.nilable(String)) }
  def content_type
    blob&.content_type
  end

  # ファイルサイズを取得（バイト単位）
  sig { returns(T.nilable(Integer)) }
  def byte_size
    blob&.byte_size
  end
end
