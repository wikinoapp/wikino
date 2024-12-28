# typed: strict
# frozen_string_literal: true

module ControllerConcerns
  module TrashedPagesSettable
    extend T::Sig
    extend ActiveSupport::Concern

    sig(:final) { void }
    private def set_trashed_pages
      cursor_paginate_page = Current.space!.pages.preload(:topic).restorable.cursor_paginate(
        after: params[:after].presence,
        before: params[:before].presence,
        limit: 100,
        order: {trashed_at: :desc, id: :desc}
      ).fetch

      @pages = cursor_paginate_page.records
      @pagination = Pagination.from_cursor_paginate(cursor_paginate_page:)
    end
  end
end
