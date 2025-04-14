# typed: strict
# frozen_string_literal: true

module Home
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      current_user = T.let(Current.viewer!, User)
      active_spaces = SpaceRepository.new.active_spaces(user: current_user)

      render Home::ShowView.new(
        active_spaces:,
        current_user_entity: current_user.to_entity
      )
    end
  end
end
