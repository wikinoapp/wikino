# typed: strict
# frozen_string_literal: true

module TopicForm
  class DestroyConfirmation < ApplicationForm
    sig { returns(T.nilable(TopicRecord)) }
    attr_accessor :topic_record

    attribute :topic_name, :string

    validates :topic_record, presence: true
    validates :topic_name, presence: true
    validate :topic_name_correct

    sig { void }
    private def topic_name_correct
      return if topic_record.nil?
      return if topic_record.not_nil!.name == topic_name

      errors.add(:topic_name, :incorrect)
    end
  end
end
