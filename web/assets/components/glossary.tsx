import { h, Fragment, Component, RenderableProps, FunctionComponent } from "preact";
import register from "preact-custom-element";
import { KNModal } from "./modal";
import getText from "../translation";

type Definition = {
	name: string;
	description: () => h.JSX.Element | string;
};

let definitions: Record<string, { en: Definition; ro: Definition }> = {
	stdin: {
		en: {
			name: "Standard input/output",
			description: () => (
				<>
					<p>Input and output data is read from the keyboard and written to the console.</p>
					<p>
						For example, in C++, submissions must use <code>std::cin</code> and <code>std::cout</code>, respectively.
					</p>
				</>
			),
		},
		ro: {
			name: "Intrare/ieșire standard",
			description: () => (
				<>
					<p> Datele de intrare și de ieșire sunt citite de la tastatură și sunt scrise în consolă. </p>
					<p>
						De exemplu, în C++, submisiile trebuie să folosească <code>std::cin</code> și <code>std::cout</code>, respectiv.
					</p>
				</>
			),
		},
	},
	lifetime_donation_amount: {
		en: {
			name: "Estimated total donation",
			description: () => (
				<>
					<p>This is a calculated projection of the amount a person has donated during their subscription</p>
					<p>The platform does not know when a payment is processed, so we try to make a best guess.</p>
				</>
			),
		},
		ro: {
			name: "Donație totală estimată",
			description: () => (
				<>
					<p>Aceasta este o proiecție calculată a sumei totale de bani pe care o persoană a donat-o prin abonament</p>
					<p>Platforma nu știe când a fost procesată o tranzacție, deci încercăm doar să facem un estimat.</p>
				</>
			),
		},
	},
	unknown: {
		en: {
			name: "Unknown definition",
			description: () => "Definition could not be loaded",
		},
		ro: {
			name: "Definiție necunoscută",
			description: () => "Definiția n-a putut fi încărcată",
		},
	},
};

export function GlossaryLink({ name, content, children }: { name: string; content: string; children: any }) {
	return (
		<a
			href=""
			onClick={(e) => {
				e.preventDefault();
				buildGlossaryModal(name);
			}}
			class="white-anchor inline-block underline decoration-dashed"
		>
			{content}
			{children}
			<i class="text-muted fas fa-fw fa-circle-info"></i>
		</a>
	);
}

// content is a stupid bodge since slotting doesn't work without shadow DOM
register(GlossaryLink, "kn-glossary", ["name", "content"]);

function GlossaryModalDOM({ name }: { name: string }) {
	if (!Object.keys(definitions).includes(name)) {
		// if pointing to unknown definition
		name = "unknown"; // unknown exists
	}
	return (
		<KNModal
			open={true}
			title={
				<span>
					{getText("glossary")}: <u>{definitions[name][window.platform_info.language].name}</u>
				</span>
			}
			large={false}
		>
			{definitions[name][window.platform_info.language].description()}
		</KNModal>
	);
}

register(GlossaryModalDOM, "kn-glossary-modal", ["name"]);

export function buildGlossaryModal(name: string) {
	const val = document.getElementById("glossary_modal_preact");
	if (val != null) {
		document.getElementById("modals")!.removeChild(val);
	}
	const newVal = document.createElement("kn-glossary-modal");
	newVal.id = "glossary_modal_preact";
	newVal.setAttribute("name", name);

	document.getElementById("modals")!.appendChild(newVal);
}
