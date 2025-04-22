# typed: true
# frozen_string_literal: true

module Topics
  module Settings
    class ShowController < ApplicationController
      include ControllerConcerns::Authenticatable
      include ControllerConcerns::Localizable

      around_action :set_locale
      before_action :require_authentication

      sig { returns(T.untyped) }
      def call
        space = SpaceRecord.find_by_identifier!(params[:space_identifier])
        space_viewer = Current.viewer!.space_viewer!(space:)
        topic = space_viewer.showable_topics.find_by!(number: params[:topic_number])
        topic_entity = topic.to_entity(space_viewer:)

        unless topic_entity.viewer_can_update?
          return render_404
        end

        render Topics::Settings::ShowView.new(
          current_user: current_user!,
          topic_entity:
        )
      end
    end
  end
end
