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
      active_space_records = current_user_record!.active_space_records
      active_spaces = SpaceRepository.new.to_models(space_records: active_space_records)

      render Home::ShowView.new(
        active_spaces:,
        current_user: current_user!
      )
    end
  end
end
