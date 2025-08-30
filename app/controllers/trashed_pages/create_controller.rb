# typed: true
# frozen_string_literal: true

module TrashedPages
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      space_member_record = current_user_record!.space_member_record(space_record:)
      page_record = space_record.find_page_by_number!(params[:page_number]&.to_i).not_nil!
      space_member_policy = SpacePolicyFactory.build(
        user_record: current_user_record!,
        space_member_record:
      )

      unless space_member_policy.can_trash_page?(page_record:)
        return render_404
      end

      Pages::TrashService.new.call(page_record:)

      flash[:notice] = t("messages.pages.moved_to_trash")
      redirect_to topic_path(space_record.identifier, page_record.topic_record.not_nil!.number)
    end
  end
end
