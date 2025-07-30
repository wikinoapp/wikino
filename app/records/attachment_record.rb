# typed: strict
# frozen_string_literal: true

class AttachmentRecord < ApplicationRecord
  self.table_name = "attachments"

  belongs_to :space, class_name: "SpaceRecord"
  belongs_to :attached_user, class_name: "UserRecord"
  belongs_to :active_storage_attachment,
    class_name: "ActiveStorage::Attachment"

  has_many :page_attachment_references,
    class_name: "PageAttachmentReferenceRecord",
    foreign_key: :attachment_id,
    dependent: :destroy

  # バリデーション
  validates :attached_at, presence: true

  # スコープ
  scope :by_space, ->(space_id) { where(space_id: space_id) }
  scope :by_user, ->(user_id) { where(attached_user_id: user_id) }
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
