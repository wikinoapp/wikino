# typed: true
# frozen_string_literal: true

module Trash
  class ShowController < ApplicationController
    include ControllerConcerns::SpaceSettable
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Authorizable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :set_current_space
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      render Views::Trash::Show.new(
        page_connection: Page.restorable_connection(before: params[:before], after: params[:after]),
        form: TrashedPagesForm.new
      )
    end
  end
end
