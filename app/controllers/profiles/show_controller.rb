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
      user_record = UserRecord.kept.find_by!(atname: params[:atname])
      joined_spaces = user_record.active_space_records.map do |space_record|
        SpaceRepository.new.to_model(space_record:)
      end
      user = UserRepository.new.to_model(user_record:)

      render Profiles::ShowView.new(current_user:, user:, joined_spaces:)
    end
  end
end
