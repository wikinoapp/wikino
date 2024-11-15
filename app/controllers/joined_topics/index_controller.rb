# typed: true
# frozen_string_literal: true

module JoinedTopics
  class IndexController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable

    layout false

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      @joined_topics = T.let(Current.user!.topics, T.nilable(Topic::PrivateRelation))
    end
  end
end
