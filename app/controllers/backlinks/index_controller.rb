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
      page_record = space_record.find_page_by_number!(params[:page_number]&.to_i)
      page_policy = PagePolicy.new(user_record: current_user_record, page_record:)

      unless page_policy.show?
        return render_404
      end

      backlink_list = PageRepository.new.backlink_list(
        user_record: current_user_record,
        page_record:,
        after: params[:after]
      )
      page = PageRepository.new.to_model(page_record:)

      render(Backlinks::IndexView.new(page:, backlink_list:), {
        content_type: "text/vnd.turbo-stream.html",
        layout: false
      })
    end
  end
end
