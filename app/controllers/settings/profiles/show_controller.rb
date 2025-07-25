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
        form = ::Profiles::EditForm.new(
          atname: current_user!.atname,
          name: current_user!.name,
          description: current_user!.description
        )

        render_component Settings::Profiles::ShowView.new(
          current_user: current_user!,
          form:
        )
      end
    end
  end
end
