# typed: strict
# frozen_string_literal: true

class PageRevision < ApplicationRecord
  belongs_to :space
  belongs_to :space_member
  belongs_to :page
end
