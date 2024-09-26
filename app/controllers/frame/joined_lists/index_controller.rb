# typed: true
# frozen_string_literal: true

module Frame
  module JoinedLists
    class IndexController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Authorizable
      include ControllerConcerns::Localizable

      layout false

      sig { returns(T.untyped) }
      def call
        @joined_lists = T.let(viewer!.lists, T.nilable(List::PrivateRelation))
      end
    end
  end
end
