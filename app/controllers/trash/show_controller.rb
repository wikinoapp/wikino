# typed: true
# frozen_string_literal: true

module Trash
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_viewer = Current.viewer!.space_viewer!(space:)

      unless space_viewer.can_view_trash?(space:)
        return render_404
      end

      page_list_entity = space.restorable_page_list_entity(
        space_viewer:,
        before: params[:before],
        after: params[:after]
      )

      render Trash::ShowView.new(
        current_user_entity: Current.viewer!.user_entity,
        space_entity: space.to_entity(space_viewer:),
        page_list_entity:,
        form: TrashedPagesForm.new
      )
    end
  end
end
