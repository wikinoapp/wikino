# typed: strict
# frozen_string_literal: true

module TopicForm
  class Creation < ApplicationForm
    include FormConcerns::TopicNameValidatable
    include FormConcerns::TopicDescriptionValidatable
    include FormConcerns::TopicVisibilityValidatable

    sig { returns(T.nilable(SpaceRecord)) }
    attr_accessor :space_record

    attribute :name, :string
    attribute :description, :string, default: ""
    attribute :visibility, :string

    validates :space_record, presence: true
    validate :name_uniqueness

    sig { void }
    private def name_uniqueness
      return if space_record.nil?
      return if name.nil?

      if space_record.not_nil!.topic_records.exists?(name:)
        errors.add(:name, :uniqueness)
      end
    end
  end
end
