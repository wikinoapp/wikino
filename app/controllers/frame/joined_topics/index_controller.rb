# typed: true
# frozen_string_literal: true

module Frame
  module JoinedTopics
    class IndexController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Authorizable
      include ControllerConcerns::Localizable

      layout false

      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        @joined_topics = T.let(Current.user!.topics, T.nilable(Topic::PrivateRelation))
      end
    end
  end
end
