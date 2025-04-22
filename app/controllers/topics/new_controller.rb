# typed: true
# frozen_string_literal: true

module Topics
  class NewController < ApplicationController
    include ControllerConcerns::Authenticatable
    include ControllerConcerns::Localizable

    around_action :set_locale
    before_action :require_authentication

    sig { returns(T.untyped) }
    def call
      space_record = SpaceRecord.find_by_identifier!(params[:space_identifier])
      current_space_member = current_user!.current_space_member(space_record:)

      unless space_viewer.can_create_topic?
        return render_404
      end

      form = NewTopicForm.new

      render Topics::NewView.new(
        current_user: current_user!,
        space_entity: space.to_entity(space_viewer:),
        form:
      )
    end
  end
end
