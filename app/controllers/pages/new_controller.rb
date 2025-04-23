# typed: true
# frozen_string_literal: true

module Pages
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user!.space_member_record(space_record:)
      topic = space.topic_records.kept.find_by!(number: params[:topic_number])

      unless space_viewer.can_create_page?(topic:)
        return render_404
      end

      result = CreateBlankedPageService.new.call(topic:)

      redirect_to edit_page_path(space.identifier, result.page.number)
    end
  end
end
