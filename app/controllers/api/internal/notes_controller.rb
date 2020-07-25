# frozen_string_literal: true

module Api
  module Internal
    class NotesController < ApplicationController
      before_action :authenticate_user!

      def index
        @note_entities = InternalApi::NoteList::FetchNotesRepository.new(graphql_client: graphql_client).call(q: params[:q])

        if @note_entities.blank?
          head :ok
        end
      end
    end
  end
end
