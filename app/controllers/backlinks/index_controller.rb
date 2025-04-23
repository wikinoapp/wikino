# typed: true
# frozen_string_literal: true

module Backlinks
  class IndexController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    layout false

    around_action :set_locale
    before_action :restore_user_session

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user!.space_member_record(space_record:)
      page = space.find_page_by_number!(params[:page_number]&.to_i)

      unless space_viewer.can_view_page?(page:)
        return render_404
      end

      backlink_list_entity = page.not_nil!.fetch_backlink_list_entity(
        space_viewer:,
        after: params[:after]
      )

      render(Backlinks::IndexView.new(
        page_entity: page.to_entity(space_viewer:),
        backlink_list_entity:
      ), {
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      })
    end
  end
end
