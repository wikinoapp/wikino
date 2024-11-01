# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module Authorizable
    extend T::Sig
    extend ActiveSupport::Concern

    included do
      include Pundit::Authorization

      sig { returns(User) }
      def pundit_user
        Current.user!
      end

      sig(:final) { void }
      private def authorize_space
        space = Space.find_by!(identifier: params[:space_identifier])
        authorize(space, :show?)
      end
    end
  end
end
