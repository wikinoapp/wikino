# typed: strict
# frozen_string_literal: true

module Welcome
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      if signed_in?
        return redirect_to(space_path(Current.user!.space.not_nil!.identifier))
      end

      render Welcome::ShowView.new
    end
  end
end
