# typed: true
# frozen_string_literal: true

module Pages
  class ShowController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

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

      link_list_entity = page.not_nil!.fetch_link_list_entity(space_viewer:)
      backlink_list_entity = page.not_nil!.fetch_backlink_list_entity(space_viewer:)

      render Pages::ShowView.new(
        current_user: current_user!,
        page_entity: page.to_entity(space_viewer:),
        link_list_entity:,
        backlink_list_entity:
      )
    end
  end
end
