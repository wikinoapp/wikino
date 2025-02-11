# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    class ShowController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable
      include ControllerConcerns::SpaceFindable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        space = find_space_by_identifier!
        form = EditSpaceForm.new(
          identifier: space.identifier,
          name: space.name
        )

        render Spaces::Settings::ShowView.new(space:, form:)
      end
    end
  end
end
