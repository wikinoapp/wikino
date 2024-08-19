# typed: strict
# frozen_string_literal: true

class Notebook < ApplicationRecord
  extend T::Sig

  include Discard::Model

  enum :visibility, {
    NotebookVisibility::Public.serialize => 0,
    NotebookVisibility::Private.serialize => 1
  }, prefix: true

  belongs_to :space
  has_many :notebook_members, dependent: :restrict_with_exception

  scope :public_only, -> { where(visibility: NotebookVisibility::Public.serialize) }
end
