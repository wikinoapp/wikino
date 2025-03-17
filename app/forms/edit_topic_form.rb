# typed: strict
# frozen_string_literal: true

class EditTopicForm < ApplicationForm
  include FormConcerns::TopicNameValidatable
  include FormConcerns::TopicDescriptionValidatable
  include FormConcerns::TopicVisibilityValidatable

  sig { returns(T.nilable(Topic)) }
  attr_accessor :topic

  attribute :name, :string
  attribute :description, :string, default: ""
  attribute :visibility, :string

  validates :topic, presence: true
  validate :name_uniqueness

  sig { returns(T.nilable(Space)) }
  def space
    topic&.space
  end

  sig { void }
  private def name_uniqueness
    return if topic.nil?
    return if name.nil?

    if space.not_nil!.topics.where.not(id: topic.not_nil!.id).exists?(name:)
      errors.add(:name, :uniqueness)
    end
  end
end
