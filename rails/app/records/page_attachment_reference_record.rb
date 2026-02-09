# typed: strict
# frozen_string_literal: true

class PageAttachmentReferenceRecord < ApplicationRecord
  self.table_name = "page_attachment_references"

  belongs_to :attachment_record, foreign_key: :attachment_id
  belongs_to :page_record, foreign_key: :page_id

  # ページに関連する添付ファイルを取得
  sig { params(page_id: Types::DatabaseId).returns(ActiveRecord::Relation) }
  def self.attachment_records_for_page(page_id:)
    preload(:attachment_record).where(page_id:)
  end
end
