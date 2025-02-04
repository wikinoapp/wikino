# typed: strict
# frozen_string_literal: true

class PageRevision < ApplicationRecord
  belongs_to :space
  belongs_to :editor, class_name: "SpaceMember"
  belongs_to :page
end
