# typed: strict
# frozen_string_literal: true

class List < ApplicationRecord
  extend T::Sig

  include Discard::Model

  enum :visibility, {
    ListVisibility::Public.serialize => 0,
    ListVisibility::Private.serialize => 1
  }, prefix: true

  belongs_to :space
  has_many :list_members, dependent: :restrict_with_exception

  scope :public_only, -> { where(visibility: ListVisibility::Public.serialize) }
end
