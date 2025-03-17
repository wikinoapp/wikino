# typed: strict
# frozen_string_literal: true

module Profiles
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      user = User.kept.find_by!(atname: params[:atname])
      joined_space_entities = user.active_spaces.map do |space|
        space.to_entity(space_viewer: Current.viewer!.space_viewer!(space:))
      end

      render Profiles::ShowView.new(
        current_user_entity: Current.viewer!.user_entity,
        user_entity: user.to_entity,
        joined_space_entities:
      )
    end
  end
end
