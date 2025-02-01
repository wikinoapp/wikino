# typed: true
# frozen_string_literal: true

module Trash
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = find_space_by_identifier!

      unless Current.viewer!.can_view_trash?(space:)
        return render_404
      end

      render Trash::ShowView.new(
        space:,
        page_connection: space.restorable_page_connection(before: params[:before], after: params[:after]),
        form: TrashedPagesForm.new
      )
    end
  end
end
