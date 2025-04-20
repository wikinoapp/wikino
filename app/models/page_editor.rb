# typed: strict
# frozen_string_literal: true

class PageEditor < ApplicationRecord
  belongs_to :space
  belongs_to :page
  belongs_to :space_member
end
