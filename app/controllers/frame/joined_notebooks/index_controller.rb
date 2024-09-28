# typed: true
# frozen_string_literal: true

module Frame
  module JoinedNotebooks
    class IndexController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Authorizable
      include ControllerConcerns::Localizable

      layout false

      sig { returns(T.untyped) }
      def call
        @joined_notebooks = T.let(viewer!.notebooks, T.nilable(Notebook::PrivateRelation))
      end
    end
  end
end
