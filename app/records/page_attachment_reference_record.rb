# typed: strict
# frozen_string_literal: true

class PageAttachmentReferenceRecord < ApplicationRecord
  extend T::Sig

  self.table_name = "page_attachment_references"

  # アソシエーション
  belongs_to :attachment, class_name: "AttachmentRecord"
  belongs_to :page, class_name: "PageRecord"

  # バリデーション
  validates :attachment_id, uniqueness: { scope: :page_id }

  # ページに関連する添付ファイルを取得
  sig { params(page_id: String).returns(ActiveRecord::Relation) }
  def self.attachments_for_page(page_id)
    includes(:attachment).where(page_id: page_id)
  end
end