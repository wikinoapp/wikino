# typed: strict
# frozen_string_literal: true

class PageRevisionRecord < ApplicationRecord
  self.table_name = "page_revisions"

  belongs_to :space
  belongs_to :space_member
  belongs_to :page
end
