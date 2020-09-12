# frozen_string_literal: true

module Api
  module Internal
    class NotesController < ApplicationController
      before_action :authenticate_user!

      def index
        note_entities = InternalApi::NoteList::NotesRepository.new(graphql_client: graphql_client).call(q: params[:q])

        render json: {
          notes: note_entities.map do |note_entity|
            {
              databaseId: note_entity.database_id,
              title: note_entity.title
            }
          end
        }
      end

      def create
        note_entity, mutation_error_entities = CreateNoteRepository.
          new(graphql_client: graphql_client).
          call(params: { body: params[:keyword] })

        render json: { databaseId: note_entity.database_id, title: note_entity.title }
      end

      def update
        note_entity = UpdateNote::NoteRepository.
          new(graphql_client: graphql_client).
          call(database_id: params[:note_id])

        @updated_note_entity, mutation_error_entities = UpdateNoteRepository.
          new(graphql_client: graphql_client).
          call(id: note_entity.id, body: note_params[:body])
      end

      private

      def note_params
        params.require(:note).permit(:body)
      end
    end
  end
end
