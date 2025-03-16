# typed: strict
# frozen_string_literal: true

class NewTopicForm < ApplicationForm
  sig { returns(T.nilable(Space)) }
  attr_accessor :space

  attribute :name, :string
  attribute :description, :string, default: ""
  attribute :visibility, :string

  validates :visibility, presence: true
  validate :name_uniqueness

  sig { void }
  private def name_uniqueness
    return if space.nil?
    return if name.nil?

    if space.not_nil!.topics.exists?(name:)
      errors.add(:name, :uniqueness)
    end
  end
end
