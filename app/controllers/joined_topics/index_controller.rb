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
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      @topics = if signed_in? && Current.space! == Current.user!.space
        T.let(Current.user!.topics.kept, T.nilable(Topic::PrivateRelation))
      else
        T.let(Current.space!.topics.kept.visibility_public, T.nilable(Topic::PrivateRelation))
      end
    end
  end
end
