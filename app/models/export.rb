# typed: strict
# frozen_string_literal: true

class Export < ApplicationRecord
  has_one_attached :file

  belongs_to :space
  belongs_to :started_by, class_name: "SpaceMember"
end
