# typed: strict
# frozen_string_literal: true

class PageRevisionRecord < ApplicationRecord
  self.table_name = "page_revisions"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :space_member_record, foreign_key: :space_member_id
  belongs_to :page_record, foreign_key: :page_id
end
