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
      active_spaces = current_user.active_spaces
      active_space_entities = active_spaces.map do |space|
        space_viewer = Current.viewer!.space_viewer!(space:)
        space.to_entity(space_viewer:)
      end

      render Home::ShowView.new(
        active_space_entities:,
        current_user_entity: current_user.to_entity
      )
    end
  end
end
