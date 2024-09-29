# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module TopicSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    private def set_topic
      @topic = T.let(viewer!.space.topics.kept.find_by!(number: params[:topic_number]), T.nilable(Topic))
      authorize(@topic, :show?)
    end
  end
end
