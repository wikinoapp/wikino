# typed: true
# frozen_string_literal: true

module Settings
  module Emails
    class ShowController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        form = ::Emails::EditForm.new

        render_component Settings::Emails::ShowView.new(
          current_user: current_user!,
          form:
        )
      end
    end
  end
end
