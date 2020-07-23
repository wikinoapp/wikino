# frozen_string_literal: true

class NotesController < ApplicationController
  before_action :authenticate_user!

  def index
    @note_entities, @page_info_entity = NoteList::FetchNotesRepository.
      new(graphql_client: graphql_client).
      fetch(
        pagination: Pagination.new(before: params[:before], after: params[:after], per: 30)
      )
  end
end
