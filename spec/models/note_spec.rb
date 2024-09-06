# typed: false
# frozen_string_literal: true

RSpec.describe Note, type: :model do
  describe "#titles_in_body" do
    it "returns titles" do
      [
        ["[[a]]", ["a"]],
        ["[[ a ]]", [" a "]],
        ["[[Hello]]", ["Hello"]],
        ["[[こんにちは✌️]]", ["こんにちは✌️"]],
        ["[[a]] [[b]]", ["a", "b"]],
        ["[[Hello]] [[World]]", ["Hello", "World"]],
        ["[[こんにちは]] [[世界🌏]]", ["こんにちは", "世界🌏"]],
        ["[ [a] ]", []],
        ["[[a]", []],
        # A bit weird, but same behavior as Obsidian, Reflect, Bear and etc.
        ["[[[a]]]", ["[a"]],
        ["[[[a]]] [[b]]", ["[a", "b"]],
        ["[[[a]]] [[[b]]]", ["[a", "[b"]],
        ["[[[ a ]]]", ["[ a "]]
      ].each do |(body, expected)|
        note = Note.new(content: NoteContent.new(body:))
        expect(note.titles_in_body).to eq(expected)
      end
    end
  end

  describe "#fetch_link_list" do
    context "記事にリンクが含まれているとき" do
      let!(:note_1) { create(:note, modified_at: Time.zone.parse("2024-01-01")) }
      let!(:note_2) { create(:note, modified_at: Time.zone.parse("2024-01-02")) }
      let!(:note_3) { create(:note, modified_at: Time.zone.parse("2024-01-03")) }
      let!(:note_4) { create(:note, linked_note_ids: [note_3.id], modified_at: Time.zone.parse("2024-01-04")) }
      let!(:target_note) { create(:note, linked_note_ids: [note_1.id, note_3.id]) }

      it "リンクの構造体を返すこと" do
        link_list = target_note.fetch_link_list

        expect(link_list.links.size).to eq(2)

        link_0 = link_list.links[0]
        expect(link_0.note).to eq(note_3)
        expect(link_0.backlinked_notes.size).to eq(1)
        expect(link_0.backlinked_notes[0]).to eq(note_4)

        link_1 = link_list.links[1]
        expect(link_1.note).to eq(note_1)
        expect(link_1.backlinked_notes.size).to eq(0)
      end
    end
  end
end
