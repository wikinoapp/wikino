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
      space = find_space_by_identifier!
      page = space.find_page_by_number!(params[:page_number]&.to_i).not_nil!

      unless Current.viewer!.can_trash_page?(page:)
        return render_404
      end

      MovePageToTrashService.new.call(page:)

      flash[:notice] = t("messages.page.moved_to_trash")
      redirect_to topic_path(space.identifier, page.topic_record.not_nil!.number)
    end
  end
end
