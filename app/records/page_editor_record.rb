# typed: strict
# frozen_string_literal: true

class PageEditorRecord < ApplicationRecord
  self.table_name = "page_editors"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :page_record, foreign_key: :page_id
  belongs_to :space_member_record, foreign_key: :space_member_id
end
