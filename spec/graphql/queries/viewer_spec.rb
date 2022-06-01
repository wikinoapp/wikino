# typed: false
# frozen_string_literal: true

describe "viewer" do
  let!(:user) { create(:user) }
  let!(:note_1) { create(:note, :with_content, user:, title: "Hello 1") }
  let!(:note_2) { create(:note, :with_content, user:, title: "Hello 2") }
  let!(:note_3) { create(:note, :with_content, user:, title: "Hello 3") }
  let!(:variables) { {q: "Hello"} }
  let!(:context) { {viewer: user} }
  let!(:query) do
    <<~GRAPHQL
      query($q: String) {
        viewer {
          notes(
            q: $q
            orderBy: { field: MODIFIED_AT, direction: DESC },
            first: 5
          ) {
            nodes {
              id
              databaseId
              title
              content {
                bodyHtml
              }
              links {
                nodes {
                  note {
                    title
                  }
                }
              }
              backlinks {
                nodes {
                  note {
                    title
                  }
                }
              }
            }
          }
        }
      }
    GRAPHQL
  end

  before do
    create(:link, note: note_1, target_note: note_2)
    create(:link, note: note_1, target_note: note_3)
  end

  it "returns notes" do
    result = NonotoSchema.execute(query, variables:, context:)

    expect(result["errors"]).to be_nil

    expected = {
      data: {
        viewer: {
          notes: {
            nodes: [
              {
                id: NonotoSchema.id_from_object(note_3),
                databaseId: note_3.id,
                title: note_3.title,
                content: {
                  bodyHtml: note_3.body_html
                },
                links: {
                  nodes: []
                },
                backlinks: {
                  nodes: [
                    {
                      note: {
                        title: note_1.title
                      }
                    }
                  ]
                }
              },
              {
                id: NonotoSchema.id_from_object(note_2),
                databaseId: note_2.id,
                title: note_2.title,
                content: {
                  bodyHtml: note_2.body_html
                },
                links: {
                  nodes: []
                },
                backlinks: {
                  nodes: [
                    {
                      note: {
                        title: note_1.title
                      }
                    }
                  ]
                }
              },
              {
                id: NonotoSchema.id_from_object(note_1),
                databaseId: note_1.id,
                title: note_1.title,
                content: {
                  bodyHtml: note_1.body_html
                },
                links: {
                  nodes: [
                    {
                      note: {
                        title: note_2.title
                      }
                    },
                    {
                      note: {
                        title: note_3.title
                      }
                    }
                  ]
                },
                backlinks: {
                  nodes: []
                }
              }
            ]
          }
        }
      }
    }
    expect(result.to_h.deep_stringify_keys).to include(expected.deep_stringify_keys)
  end
end
