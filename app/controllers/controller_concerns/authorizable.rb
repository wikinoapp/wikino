# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authorizable
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      include Pundit::Authorization

      sig { returns(T.nilable(User)) }
      def pundit_user
        Current.user
      end
    end
  end
end
