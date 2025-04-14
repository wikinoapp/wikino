# typed: strict
# frozen_string_literal: true

class UpdateTopicService < ApplicationService
  class Result < T::Struct
    const :topic, TopicRecord
  end

  sig { params(form: EditTopicForm).returns(Result) }
  def call(form:)
    topic = form.topic.not_nil!
    topic.attributes = {
      name: form.name.not_nil!,
      description: form.description.not_nil!,
      visibility: form.visibility.not_nil!
    }

    topic.save!

    Result.new(topic:)
  end
end
