# frozen_string_literal: true

module Api
  module Internal
    class NotesController < ApplicationController
      before_action :authenticate_user!

      def index
        @note_entities = InternalApi::NoteList::FetchNotesRepository.new(graphql_client: graphql_client).call(q: params[:q])
      end

      def create
        note_entity, mutation_error_entities = CreateNoteRepository.
          new(graphql_client: graphql_client).
          call(params: { body: params[:note_title] })

        render json: { database_id: note_entity.database_id, title: note_entity.title }
      end
    end
  end
end
