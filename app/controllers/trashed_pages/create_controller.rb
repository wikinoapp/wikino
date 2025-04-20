# typed: true
# frozen_string_literal: true

module TrashedPages
  class CreateController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable
    include ControllerConcerns::SpaceFindable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = find_space_by_identifier!
      page_record = space_record.find_page_by_number!(params[:page_number]&.to_i).not_nil!
      page_policy = Page::PolicyRepository.new.build(
        user: Current.viewer!,
        page: PageRepository.new.build_model(page_record:)
      )

      unless page_policy.can_trash?
        return render_404
      end

      MovePageToTrashService.new.call(page: page_record)

      flash[:notice] = t("messages.page.moved_to_trash")
      redirect_to topic_path(space_record.identifier, page_record.topic_record.not_nil!.number)
    end
  end
end
