# typed: strict
# frozen_string_literal: true

class SpaceMemberPermission < T::Enum
  enums do
    CreateTopic = new("create:topic")
    CreatePage = new("create:page")
    CreateDraftPage = new("create:draft_page")
    ExportSpace = new("export:space")
    UpdateSpace = new("update:space")
    UpdateTopic = new("update:topic")
  end
end
