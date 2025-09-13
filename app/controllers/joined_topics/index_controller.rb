# typed: strict
# frozen_string_literal: true

module JoinedTopics
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      topics = TopicRepository.new.find_joined_topics(user_record: current_user_record!, limit: 10)
      variant = params[:variant]&.to_sym || :fixed

      render_component(JoinedTopics::IndexView.new(
        topics:,
        variant:
      ))
    end
  end
end
