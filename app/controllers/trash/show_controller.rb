# typed: true
# frozen_string_literal: true

module Trash
  class ShowController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable
    include ControllerConcerns::TrashedPagesSettable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication
    before_action :set_trashed_pages

    sig { returns(T.untyped) }
    def call
      @form = TrashedPagesForm.new
    end
  end
end
