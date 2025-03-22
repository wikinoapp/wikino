# typed: strict
# frozen_string_literal: true

module Settings
  module Profiles
    class ShowController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        current_user = T.let(Current.viewer!, User)
        form = EditProfileForm.new(
          atname: current_user.atname,
          name: current_user.name,
          description: current_user.description
        )

        render Settings::Profiles::ShowView.new(
          current_user_entity: current_user.to_entity,
          form:
        )
      end
    end
  end
end
