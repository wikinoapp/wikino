# typed: strict
# frozen_string_literal: true

module Welcome
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale

    sig { returns(T.untyped) }
    def call
      restore_session

      if signed_in?
        redirect_to(space_path(Current.user!.space.not_nil!.identifier))
      end
    end
  end
end
