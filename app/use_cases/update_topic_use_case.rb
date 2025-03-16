# typed: strict
# frozen_string_literal: true

class UpdateTopicUseCase < ApplicationUseCase
  class Result < T::Struct
    const :topic, Topic
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
