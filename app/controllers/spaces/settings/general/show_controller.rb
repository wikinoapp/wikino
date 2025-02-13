# typed: true
# frozen_string_literal: true

module Spaces
  module Settings
    module General
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

          render Spaces::Settings::General::ShowView.new(space:, form:)
        end
      end
    end
  end
end
