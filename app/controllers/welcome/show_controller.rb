# typed: strict
# frozen_string_literal: true

module Welcome
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale

    sig { returns(T.untyped) }
    def call
      if signed_in?
        redirect_to(space_path(space_identifier: Current.space!.identifier))
      end
    end
  end
end
