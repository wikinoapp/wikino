# typed: true
# frozen_string_literal: true

module Topics
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TopicSettable

    around_action :set_locale
    before_action :require_authentication
    before_action :set_topic

    sig { returns(T.untyped) }
    def call
      @notes = @topic.not_nil!.notes.published
    end
  end
end
