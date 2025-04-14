# typed: strict
# frozen_string_literal: true

class PageEditorRecord < ApplicationRecord
  self.table_name = "page_editors"

  belongs_to :space
  belongs_to :page
  belongs_to :space_member
end
