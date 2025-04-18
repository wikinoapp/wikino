# typed: strict
# frozen_string_literal: true

class EditTopicForm < ApplicationForm
  include FormConcerns::TopicNameValidatable
  include FormConcerns::TopicDescriptionValidatable
  include FormConcerns::TopicVisibilityValidatable

  sig { returns(T.nilable(TopicRecord)) }
  attr_accessor :topic_record

  attribute :name, :string
  attribute :description, :string, default: ""
  attribute :visibility, :string

  validates :topic_record, presence: true
  validate :name_uniqueness

  sig { returns(T.nilable(SpaceRecord)) }
  def space_record
    topic_record&.space_record
  end

  sig { void }
  private def name_uniqueness
    return if topic_record.nil?
    return if name.nil?

    if space_record.not_nil!.topic_records.where.not(id: topic_record.not_nil!.id).exists?(name:)
      errors.add(:name, :uniqueness)
    end
  end
end
